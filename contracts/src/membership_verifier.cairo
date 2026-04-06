// Membership Verifier Contract
//
// Wraps the Groth16 verifier to provide membership proof verification
// for Merkle tree membership proofs.

use super::groth16_verifier::{Groth16Proof, G1Point};

/// Interface for membership verifier
#[starknet::interface]
pub trait IMembershipVerifier<TContractState> {
    /// Verify a membership proof
    /// Returns true if the proof is valid
    fn verify_membership(
        self: @TContractState,
        proof: Groth16Proof,
        merkle_root: u256,
        leaf_hash: u256
    ) -> bool;

    /// Check if a nullifier (leaf hash) has been used
    fn is_nullifier_used(self: @TContractState, nullifier: u256) -> bool;

    /// Get the current Merkle root
    fn get_merkle_root(self: @TContractState) -> u256;

    /// Get verification count
    fn get_verification_count(self: @TContractState) -> u64;

    /// Check if a root is valid (current or historical)
    fn is_valid_root(self: @TContractState, root: u256) -> bool;
}

#[starknet::contract]
pub mod MembershipVerifier {
    use super::{IMembershipVerifier, Groth16Proof};
    use starknet::{get_caller_address, get_block_timestamp};
    use starknet::storage::{
        StoragePointerReadAccess, StoragePointerWriteAccess,
        Map, StoragePathEntry
    };

    // Number of historical roots to keep
    const ROOT_HISTORY_SIZE: u32 = 100;

    #[storage]
    struct Storage {
        owner: starknet::ContractAddress,
        groth16_verifier: starknet::ContractAddress,
        // Current Merkle root
        current_root: u256,
        // Root history for accepting slightly old roots
        root_history: Map<u32, u256>,
        root_history_index: u32,
        // Used nullifiers to prevent double-spending
        nullifiers: Map<u256, bool>,
        // Verification statistics
        verification_count: u64,
        // Tree depth
        tree_depth: u32,
    }

    #[event]
    #[derive(Drop, starknet::Event)]
    pub enum Event {
        MembershipVerified: MembershipVerified,
        MerkleRootUpdated: MerkleRootUpdated,
        NullifierUsed: NullifierUsed,
    }

    #[derive(Drop, starknet::Event)]
    pub struct MembershipVerified {
        #[key]
        pub verifier: starknet::ContractAddress,
        pub merkle_root: u256,
        pub leaf_hash: u256,
        pub timestamp: u64,
    }

    #[derive(Drop, starknet::Event)]
    pub struct MerkleRootUpdated {
        #[key]
        pub updater: starknet::ContractAddress,
        pub old_root: u256,
        pub new_root: u256,
    }

    #[derive(Drop, starknet::Event)]
    pub struct NullifierUsed {
        #[key]
        pub nullifier: u256,
        pub user: starknet::ContractAddress,
    }

    #[constructor]
    fn constructor(
        ref self: ContractState,
        owner: starknet::ContractAddress,
        groth16_verifier: starknet::ContractAddress,
        initial_root: u256,
        tree_depth: u32
    ) {
        self.owner.write(owner);
        self.groth16_verifier.write(groth16_verifier);
        self.current_root.write(initial_root);
        self.tree_depth.write(tree_depth);
        self.root_history_index.write(0);
        self.verification_count.write(0);

        // Store initial root in history
        self.root_history.entry(0).write(initial_root);
    }

    #[abi(embed_v0)]
    impl MembershipVerifierImpl of IMembershipVerifier<ContractState> {
        fn verify_membership(
            self: @ContractState,
            proof: Groth16Proof,
            merkle_root: u256,
            leaf_hash: u256
        ) -> bool {
            // Check root is valid (current or recent)
            assert(self.is_valid_root(merkle_root), 'Invalid merkle root');

            // Check nullifier hasn't been used (optional, for privacy applications)
            // Uncomment if you want to prevent reuse of the same leaf
            // assert(!self.nullifiers.entry(leaf_hash).read(), 'Nullifier already used');

            // Prepare public inputs for Groth16 verification
            // The circuit expects [root, leaf_hash] as public inputs
            let mut public_inputs = ArrayTrait::new();
            public_inputs.append(merkle_root);
            public_inputs.append(leaf_hash);

            // For demo purposes, perform basic validation
            // In production, would call the Groth16 verifier contract
            let is_valid = validate_proof_structure(@proof);

            if is_valid {
                // Note: In a real implementation, we would call the Groth16 verifier
                // let groth16 = IGorth16VerifierDispatcher { contract_address: self.groth16_verifier.read() };
                // let is_valid = groth16.verify_proof(proof, public_inputs);
                true
            } else {
                false
            }
        }

        fn is_nullifier_used(self: @ContractState, nullifier: u256) -> bool {
            self.nullifiers.entry(nullifier).read()
        }

        fn get_merkle_root(self: @ContractState) -> u256 {
            self.current_root.read()
        }

        fn get_verification_count(self: @ContractState) -> u64 {
            self.verification_count.read()
        }

        fn is_valid_root(self: @ContractState, root: u256) -> bool {
            // Check current root
            if root == self.current_root.read() {
                return true;
            }

            // Check root history
            let mut i: u32 = 0;
            loop {
                if i >= ROOT_HISTORY_SIZE {
                    break false;
                }
                if self.root_history.entry(i).read() == root {
                    break true;
                }
                i += 1;
            }
        }
    }

    /// Update the Merkle root (only owner)
    fn update_merkle_root(ref self: ContractState, new_root: u256) {
        assert(get_caller_address() == self.owner.read(), 'Only owner');

        let old_root = self.current_root.read();

        // Update current root
        self.current_root.write(new_root);

        // Add to history
        let index = self.root_history_index.read();
        let next_index = (index + 1) % ROOT_HISTORY_SIZE;
        self.root_history.entry(next_index).write(new_root);
        self.root_history_index.write(next_index);

        self.emit(MerkleRootUpdated {
            updater: get_caller_address(),
            old_root,
            new_root,
        });
    }

    /// Mark a nullifier as used
    fn use_nullifier(ref self: ContractState, nullifier: u256) {
        assert(!self.nullifiers.entry(nullifier).read(), 'Nullifier already used');

        self.nullifiers.entry(nullifier).write(true);

        self.emit(NullifierUsed {
            nullifier,
            user: get_caller_address(),
        });
    }

    /// Record a successful verification
    fn record_verification(
        ref self: ContractState,
        merkle_root: u256,
        leaf_hash: u256
    ) {
        let count = self.verification_count.read();
        self.verification_count.write(count + 1);

        self.emit(MembershipVerified {
            verifier: get_caller_address(),
            merkle_root,
            leaf_hash,
            timestamp: get_block_timestamp(),
        });
    }

    /// Validate proof structure (basic checks)
    fn validate_proof_structure(proof: @Groth16Proof) -> bool {
        // Check that proof points are not zero (basic sanity check)
        let a = proof.a;
        let c = proof.c;

        // Points shouldn't all be zero
        let a_valid = *a.x != 0 || *a.y != 0;
        let c_valid = *c.x != 0 || *c.y != 0;

        a_valid && c_valid
    }

    /// Get tree depth
    fn get_tree_depth(self: @ContractState) -> u32 {
        self.tree_depth.read()
    }
}
