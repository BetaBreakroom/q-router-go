package fibo

import "errors"

func Fibo(n int) (int, error) {
	if n < 0 {
		return 0, errors.New("fibo: negative input")
	}
	if n == 0 {
		return 0, nil
	}
	if n == 1 {
		return 1, nil
	}

	prev2 := 0
	prev1 := 1
	for i := 2; i <= n; i++ {
		curr := prev1 + prev2
		prev2 = prev1
		prev1 = curr
	}

	return prev1, nil
}