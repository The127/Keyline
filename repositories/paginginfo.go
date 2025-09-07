package repositories

type pagingInfo struct {
	page int
	size int
}

func (i pagingInfo) offset() int {
	return (i.page - 1) * i.size
}
