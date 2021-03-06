// Code generated by "stringer -type=Type"; DO NOT EDIT.

package trigger

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Verification-0]
	_ = x[VerificationR-1]
	_ = x[Application-16]
	_ = x[ApplicationR-17]
}

const (
	_Type_name_0 = "VerificationVerificationR"
	_Type_name_1 = "ApplicationApplicationR"
)

var (
	_Type_index_0 = [...]uint8{0, 12, 25}
	_Type_index_1 = [...]uint8{0, 11, 23}
)

func (i Type) String() string {
	switch {
	case i <= 1:
		return _Type_name_0[_Type_index_0[i]:_Type_index_0[i+1]]
	case 16 <= i && i <= 17:
		i -= 16
		return _Type_name_1[_Type_index_1[i]:_Type_index_1[i+1]]
	default:
		return "Type(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
