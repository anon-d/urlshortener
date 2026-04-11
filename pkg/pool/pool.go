package pool

// Интерфейс ограничение
type Resettable interface {
	Reset()
}

// Pool с generic
type Pool[T Resettable] struct {
	items []T
	new   func() T // подглядел у sync.Pool :)
}

// New - конструктор *Pool[T]. cap — вместимость пула, factory — функция создания нового объекта
func New[T Resettable](cap int, factory func() T) *Pool[T] {
	return &Pool[T]{
		items: make([]T, 0, cap),
		new:   factory,
	}
}

// Put — кладет объект в пул, предварительно сбросив его
func (p *Pool[T]) Put(item T) {
	item.Reset()
	p.items = append(p.items, item)
}

// Get — возвращает объект из пула
// если запрашиваем объект, которого нет в пуле, создаем его через factory,
// т.к. мы не можем запрашивать то, чего нет
func (p *Pool[T]) Get() T {
	n := len(p.items)

	if n == 0 {
		return p.new()
	}

	item := p.items[n-1]
	p.items = p.items[:n-1]

	return item
}
