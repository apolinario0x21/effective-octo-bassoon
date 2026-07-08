package api

// isValidCPF verifica se o CPF tem 11 dígitos numéricos e dígitos
// verificadores corretos, rejeitando sequências de dígitos repetidos.
func isValidCPF(cpf string) bool {
	if len(cpf) != 11 {
		return false
	}

	digits := make([]int, 11)
	allEqual := true

	for i, r := range cpf {
		if r < '0' || r > '9' {
			return false
		}

		digits[i] = int(r - '0')
		if digits[i] != digits[0] {
			allEqual = false
		}
	}

	if allEqual {
		return false
	}

	for _, position := range []int{9, 10} {
		sum := 0
		for i := 0; i < position; i++ {
			sum += digits[i] * (position + 1 - i)
		}

		if digits[position] != (sum*10)%11%10 {
			return false
		}
	}

	return true
}
