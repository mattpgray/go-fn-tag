package testdir

type TestType struct{}

// TestType1 is a good function with old style fn tags
func (t *TestType) TestType1() string {
	fn := "testdir.method.TestType1"
	return fn
} // TestType1

// TestType2 is a good function with new style fn tags
func (t *TestType) TestType2() string {
	fn := "testdir.method.*TestType-TestType2"
	return fn
} // TestType2

func (t TestType) TestType3() string {
	fn := "testdir.method.TestType-TestType3"
	return fn
} // TestType3

func (t *TestType) TestTypeBad1() string {
	fn := "testdir.method.ttt"
	return fn
} // TestType3

func (t TestType) TestTypeBad2() string {
	fn := "testdir.method.ttt"
	return fn
} // TestType3
