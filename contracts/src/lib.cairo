// ZKP Verifier Contracts for Starknet/Katana
//
// This library provides:
// - Groth16 proof verification for BN254 curve
// - Membership proof verification with Merkle tree support

pub mod groth16_verifier;
pub mod membership_verifier;

// Re-export main interfaces
pub use groth16_verifier::{
    IGorth16Verifier, IGorth16VerifierDispatcher, IGorth16VerifierDispatcherTrait,
    Groth16Proof, VerificationKey, G1Point, G2Point
};

pub use membership_verifier::{
    IMembershipVerifier, IMembershipVerifierDispatcher, IMembershipVerifierDispatcherTrait
};
