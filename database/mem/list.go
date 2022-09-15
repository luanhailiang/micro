package mem

import "sync"

type ItemList struct {
	lock  sync.RWMutex
	Items map[string]*Item
}

func NewItemList() *ItemList {
	list := &ItemList{
		Items: map[string]*Item{},
	}
	return list
}

func (p *ItemList) GetItem(index string) *Item {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.Items[index]
}

func (p *ItemList) SetItem(index string, item *Item) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.Items[index] = item
}

func (p *ItemList) DelItem(index string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.Items, index)
}

func (p *ItemList) Clear(over int64) int {
	p.lock.Lock()
	defer p.lock.Unlock()

	var count int
	for index, item := range p.Items {
		if !item.TimeOut(over) {
			continue
		}
		delete(p.Items, index)
		count++
	}
	return count
}
