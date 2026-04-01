// Package compliance provides data classification utilities.
package compliance

// DataClass classifies the sensitivity of data.
type DataClass int

const (
	// DataClassPublic is non-sensitive data with no access restrictions.
	DataClassPublic DataClass = iota
	// DataClassInternal is internal data accessible to org members.
	DataClassInternal
	// DataClassConfidential is sensitive data requiring explicit access grants.
	DataClassConfidential
	// DataClassPHI is Protected Health Information under HIPAA § 164.312.
	DataClassPHI
)

// IsPHI reports whether dc is Protected Health Information.
func IsPHI(dc DataClass) bool {
	return dc == DataClassPHI
}

// RequiresEncryption reports whether dc must be encrypted at rest.
func RequiresEncryption(dc DataClass) bool {
	return dc >= DataClassConfidential
}
