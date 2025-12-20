package fibo

import "errors"

func Fibo(n int) (int, error) {
	if n < 0 {
		return -1, errors.New("n must be non-negative")
	}
	switch n {
	case 0:
		return 0, nil
	case 1:
		return 1, nil
	default:
		res1, err1 := Fibo(n-1)
		if err1 != nil {
			return -1, err1
		}
		res2, err2 := Fibo(n-2)
		if err2 != nil {
			return -1, err2
		}
		return res1 + res2, nil
	}
}