//go:build !windows

package initiative

func readMachine() Machine {
	return Machine{}
}
