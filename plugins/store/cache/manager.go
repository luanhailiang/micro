package mem

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/luanhailiang/micro.git/plugins/matedata"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

var db *Manager
var once sync.Once

// LOCAL_CACHE 必须存在
func GetManager() *Manager {
	once.Do(func() {
		db = NewManager()
		over, err := strconv.Atoi(os.Getenv("LOCAL_CACHE"))
		if err != nil {
			logrus.Fatalf("LOCAL_CACHE:%s %s", os.Getenv("LOCAL_CACHE"), err.Error())
		}
		db.over = int64(over)
	})
	return db
}

func NewManager() *Manager {
	m := &Manager{}
	m.name = os.Getenv("SERVE_NAME")
	if os.Getenv("NO_SPACE") == "" {
		m.zone = true
	}
	m.over = 60 //默认1分钟
	m.list = make([]*ItemList, hash_len)
	for i := 0; i < hash_len; i++ {
		m.list[i] = NewItemList()
	}
	m.heart()
	return m
}

type Manager struct {
	name string
	zone bool
	over int64
	list []*ItemList
}

func (p *Manager) SetOver(over int64) {
	p.over = over
}

func (p *Manager) heart() {
	go func() {
		logrus.Info("mem heart beat start ", p.over)
		tick := time.Tick(time.Duration(p.over) * time.Second)
		for t := range tick {
			var sum int
			var count int
			for _, list := range p.list {
				sum += len(list.Items)
				count += list.Clear(p.over)

			}
			logrus.Infof("mem heart clear all %d del %d use %dms", sum, count, time.Now().UnixMilli()-t.UnixMilli())
		}
	}()
}

func (p *Manager) index(ctx context.Context, table, id string) string {
	if !p.zone {
		return fmt.Sprintf("%s_%s", table, id)
	}
	mate, ok := matedata.FromMateContext(ctx)
	if !ok {
		logrus.Errorf("keyCtx lost mate")
	}
	// if mate.Space == "" {
	// 	return fmt.Sprintf("%s_%s", table, id)
	// }
	if len(mate.Space) != 4 {
		num, err := strconv.Atoi(mate.Space)
		if err == nil {
			mate.Space = fmt.Sprintf("%04d", num)
		}
	}
	return fmt.Sprintf("%s_%s_%s", table, mate.Space, id)
}

func (p *Manager) GetItem(index string) *Item {
	i := hash(index)
	return p.list[i].GetItem(index)
}

func (p *Manager) SetItem(index string, item *Item) {
	i := hash(index)
	p.list[i].SetItem(index, item)
}
func (p *Manager) DelItem(index string) {
	i := hash(index)
	p.list[i].DelItem(index)
}

func (p *Manager) GetOne(ctx context.Context, id string, obj proto.Message) (err error) {
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())

	index := p.index(ctx, table, id)
	item := p.GetItem(index)
	if item == nil {
		return fmt.Errorf("item not exist")
	}
	if item.TimeOut(p.over) {
		return fmt.Errorf("item time out")
	}
	item.Load(obj)
	return nil
}

func (p *Manager) SetOne(ctx context.Context, id string, obj proto.Message) (err error) {
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())

	item := &Item{}
	item.Dump(obj)
	index := p.index(ctx, table, id)
	p.SetItem(index, item)
	return err
}

func (p *Manager) DelOne(ctx context.Context, id string, obj proto.Message) {
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())

	index := p.index(ctx, table, id)
	p.DelItem(index)
}

func (p *Manager) GetAll(ctx context.Context, ids []string, results any) ([]string, error) {
	resultsVal := reflect.ValueOf(results)
	if resultsVal.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("results argument must be a pointer to a slice, but was a %s", resultsVal.Kind())
	}

	sliceVal := resultsVal.Elem()
	if sliceVal.Kind() == reflect.Interface {
		sliceVal = sliceVal.Elem()
	}

	if sliceVal.Kind() != reflect.Slice {
		return nil, fmt.Errorf("results argument must be a pointer to a slice, but was a pointer to %s", sliceVal.Kind())
	}
	list := []string{}
	elementType := sliceVal.Type().Elem()
	for _, id := range ids {
		newElem := reflect.New(elementType)

		msg := newElem.Addr().Interface().(proto.Message)
		err := p.GetOne(ctx, id, msg)
		if err == nil {
			sliceVal = reflect.Append(sliceVal, newElem.Elem())
		} else {
			// sliceVal = reflect.Append(sliceVal, reflect.ValueOf(nil).Elem())
			list = append(list, id)
		}
		sliceVal = sliceVal.Slice(0, sliceVal.Cap())
	}
	resultsVal.Elem().Set(sliceVal.Slice(0, sliceVal.Cap()))
	return list, nil
}
