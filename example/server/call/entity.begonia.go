// Code generated by Begonia. DO NOT EDIT.
// versions:
// 	Begonia v1.0.2
// begonia call entity file

package call

type TestStruct struct {
	I1 int
	I2 int8
	I3 int16
	I4 int32
	I5 int64

	Str string
	S1  []int
	S2  []string

	TestStruct2
	Test3 TestStruct2

	Map1 map[string]string
	Map2 map[string][]int
}

type TestStruct2 struct {
	B1 []byte
	B2 []uint8
}
