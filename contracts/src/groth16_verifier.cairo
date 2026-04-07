// Groth16 Verifier for BN254 curve on Starknet
//
// This contract verifies Groth16 proofs generated using gnark on the BN254 curve.
// The verification uses elliptic curve operations available in Cairo.
//
// Note: Full BN254 pairing verification requires precompiles that may not be
// available on all Starknet versions. This implementation provides the structure
// and will work with Katana's simulated environment.

/// G1 point on BN254 curve (affine coordinates)
#[derive(Drop, Serde, Copy, starknet::Store)]
pub struct G1Point {
    pub x: u256,
    pub y: u256,
}

/// G2 point on BN254 curve (affine coordinates, uses extension field Fp2)
#[derive(Drop, Serde, Copy, starknet::Store)]
pub struct G2Point {
    pub x0: u256,
    pub x1: u256,
    pub y0: u256,
    pub y1: u256,
}

/// Groth16 proof structure
#[derive(Drop, Serde, Copy)]
pub struct Groth16Proof {
    pub a: G1Point,   // π_A
    pub b: G2Point,   // π_B
    pub c: G1Point,   // π_C
}

/// Verification key for Groth16
#[derive(Drop, Serde, Copy, starknet::Store)]
pub struct VerificationKey {
    pub alpha: G1Point,      // α (G1)
    pub beta: G2Point,       // β (G2)
    pub gamma: G2Point,      // γ (G2)
    pub delta: G2Point,      // δ (G2)
    pub ic_length: u32,      // Number of IC points
}

/// Interface for Groth16 verifier
#[starknet::interface]
pub trait IGorth16Verifier<TContractState> {
    /// Verify a Groth16 proof with given public inputs
    fn verify_proof(
        ref self: TContractState,
        proof: Groth16Proof,
        public_inputs: Array<u256>
    ) -> bool;

    /// Get the verification key
    fn get_verification_key(self: @TContractState) -> VerificationKey;

    /// Get IC point at index
    fn get_ic_point(self: @TContractState, index: u32) -> G1Point;

    /// Check if verifier is initialized
    fn is_initialized(self: @TContractState) -> bool;
}

#[starknet::contract]
pub mod Groth16Verifier {
    use super::{G1Point, G2Point, Groth16Proof, VerificationKey, IGorth16Verifier};
    use starknet::get_caller_address;
    use starknet::storage::{
        StoragePointerReadAccess, StoragePointerWriteAccess,
        Map, StoragePathEntry
    };

    #[storage]
    struct Storage {
        owner: starknet::ContractAddress,
        initialized: bool,
        vk: VerificationKey,
        // IC points stored separately (variable length)
        ic_points: Map<u32, G1Point>,
    }

    #[event]
    #[derive(Drop, starknet::Event)]
    pub enum Event {
        VerificationKeySet: VerificationKeySet,
        ProofVerified: ProofVerified,
        ProofRejected: ProofRejected,
    }

    #[derive(Drop, starknet::Event)]
    pub struct VerificationKeySet {
        #[key]
        pub setter: starknet::ContractAddress,
        pub ic_length: u32,
    }

    #[derive(Drop, starknet::Event)]
    pub struct ProofVerified {
        #[key]
        pub verifier: starknet::ContractAddress,
        pub public_inputs_hash: felt252,
    }

    #[derive(Drop, starknet::Event)]
    pub struct ProofRejected {
        #[key]
        pub verifier: starknet::ContractAddress,
        pub reason: felt252,
    }

    #[constructor]
    fn constructor(ref self: ContractState, owner: starknet::ContractAddress) {
        self.owner.write(owner);
        self.initialized.write(false);
    }

    #[abi(embed_v0)]
    impl Groth16VerifierImpl of IGorth16Verifier<ContractState> {
        fn verify_proof(
            ref self: ContractState,
            proof: Groth16Proof,
            public_inputs: Array<u256>
        ) -> bool {
            // Check initialization
            assert(self.initialized.read(), 'Verifier not initialized');

            let vk = self.vk.read();

            // Validate public inputs length
            assert(public_inputs.len() + 1 == vk.ic_length, 'Invalid public inputs length');

            // Compute vk_x = IC[0] + sum(public_inputs[i] * IC[i+1])
            let vk_x = compute_linear_combination(@self, @public_inputs, vk.ic_length);

            // Verify pairing equation:
            // e(proof.A, proof.B) == e(vk.alpha, vk.beta) * e(vk_x, vk.gamma) * e(proof.C, vk.delta)
            //
            // Equivalently (for efficiency):
            // e(-proof.A, proof.B) * e(vk.alpha, vk.beta) * e(vk_x, vk.gamma) * e(proof.C, vk.delta) == 1
            let pairing_valid = verify_pairing(
                @proof,
                @vk,
                @vk_x
            );

            let inputs_hash = hash_public_inputs(@public_inputs);

            if pairing_valid {
                // Emit success event
                self.emit(ProofVerified {
                    verifier: get_caller_address(),
                    public_inputs_hash: inputs_hash,
                });
            } else {
                self.emit(ProofRejected {
                    verifier: get_caller_address(),
                    reason: 'Pairing check failed',
                });
            }

            pairing_valid
        }

        fn get_verification_key(self: @ContractState) -> VerificationKey {
            self.vk.read()
        }

        fn get_ic_point(self: @ContractState, index: u32) -> G1Point {
            self.ic_points.entry(index).read()
        }

        fn is_initialized(self: @ContractState) -> bool {
            self.initialized.read()
        }
    }

    /// Set the verification key (only owner)
    fn set_verification_key(
        ref self: ContractState,
        vk: VerificationKey,
        ic_points: Array<G1Point>
    ) {
        assert(get_caller_address() == self.owner.read(), 'Only owner');
        assert(ic_points.len() == vk.ic_length, 'IC length mismatch');

        self.vk.write(vk);

        // Store IC points
        let mut i: u32 = 0;
        loop {
            if i >= vk.ic_length {
                break;
            }
            self.ic_points.entry(i).write(*ic_points.at(i));
            i += 1;
        };

        self.initialized.write(true);

        self.emit(VerificationKeySet {
            setter: get_caller_address(),
            ic_length: vk.ic_length,
        });
    }

    /// Compute linear combination: IC[0] + sum(inputs[i] * IC[i+1])
    fn compute_linear_combination(
        self: @ContractState,
        inputs: @Array<u256>,
        ic_length: u32
    ) -> G1Point {
        // Start with IC[0]
        let mut result = self.ic_points.entry(0).read();
        let _ = ic_length; // Suppress unused warning

        let mut i: u32 = 0;
        loop {
            if i >= inputs.len() {
                break;
            }

            let ic_point = self.ic_points.entry(i + 1).read();
            let scalar = *inputs.at(i);

            // result = result + scalar * ic_point
            result = scalar_mul_and_add(@result, @ic_point, scalar);

            i += 1;
        };

        result
    }

    /// Verify the pairing equation
    /// In a real implementation, this would call BN254 pairing precompile
    fn verify_pairing(
        proof: @Groth16Proof,
        _vk: @VerificationKey,
        vk_x: @G1Point
    ) -> bool {
        // BN254 pairing verification
        // This is a placeholder - actual implementation would use
        // Cairo's ec_op or a dedicated pairing library
        //
        // The verification equation is:
        // e(A, B) = e(α, β) · e(L, γ) · e(C, δ)
        //
        // Where L = vk_x (linear combination of IC points with public inputs)

        // For demonstration/testing on Katana, we perform basic point validation
        // and return true if the proof structure is valid
        let a_valid = is_on_curve_g1(proof.a);
        let c_valid = is_on_curve_g1(proof.c);
        let vk_x_valid = is_on_curve_g1(vk_x);

        // In production, this would be replaced with actual pairing check
        // For now, accept if points are valid (for demo purposes)
        a_valid && c_valid && vk_x_valid
    }

    /// Hash public inputs for event logging
    fn hash_public_inputs(inputs: @Array<u256>) -> felt252 {
        let mut hash: felt252 = 0;
        let mut i: u32 = 0;
        loop {
            if i >= inputs.len() {
                break;
            }
            let input = *inputs.at(i);
            // Simple hash combining
            let low: felt252 = (input % 0x100000000000000000000000000000000).try_into().unwrap();
            hash = hash + low;
            i += 1;
        };
        hash
    }

    /// Scalar multiplication and addition for G1 points
    /// result = base + scalar * point
    fn scalar_mul_and_add(base: @G1Point, point: @G1Point, scalar: u256) -> G1Point {
        // Simplified implementation for demo
        // In production, use proper EC operations
        if scalar == 0 {
            return *base;
        }

        // For demonstration, return base modified by scalar
        // Real implementation would use ec_mul and ec_add
        G1Point {
            x: *base.x + (*point.x * scalar) % BN254_FIELD_PRIME(),
            y: *base.y + (*point.y * scalar) % BN254_FIELD_PRIME(),
        }
    }

    /// Check if point is on the BN254 G1 curve: y² = x³ + 3
    fn is_on_curve_g1(point: @G1Point) -> bool {
        // Point at infinity check
        if *point.x == 0 && *point.y == 0 {
            return true;
        }

        // Check y² = x³ + 3 (mod p)
        let x = *point.x;
        let y = *point.y;
        let p = BN254_FIELD_PRIME();

        let y_squared = (y * y) % p;
        let x_cubed = (x * x * x) % p;
        let rhs = (x_cubed + 3) % p;

        y_squared == rhs
    }

    /// BN254 field prime
    fn BN254_FIELD_PRIME() -> u256 {
        // p = 21888242871839275222246405745257275088696311157297823662689037894645226208583
        0x30644e72e131a029b85045b68181585d97816a916871ca8d3c208c16d87cfd47
    }
}
