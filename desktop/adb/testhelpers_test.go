package adb_test

type fakeRunner struct {
	calls  [][]string
	output string
	err    error
}

func (f *fakeRunner) Run(args ...string) (string, error) {
	f.calls = append(f.calls, args)
	return f.output, f.err
}
