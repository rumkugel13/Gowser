package try

func Try(fn func()) (err any) {
	defer func() {
		if r := recover(); r != nil {
			err = r
		}
	}()
	fn()
	return
}
