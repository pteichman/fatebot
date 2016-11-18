package main

func Squish(s string, max int) string {
	var (
		ret []rune
		cur rune
		n   int
	)

	emit := func(r rune, n int) {
		if n > max {
			n = max
		}
		for i := 0; i < n; i++ {
			ret = append(ret, r)
		}
	}

	for _, r := range s {
		if r == cur {
			n++
		} else {
			emit(cur, n)
			cur = r
			n = 1
		}
	}

	emit(cur, n)
	return string(ret)
}

