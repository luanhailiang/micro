package rds

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
)

func SetOne(ctx context.Context, id string, obj proto.Message) (err error) {
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())

	client := GetClient()
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	return client.HSet(ctx, Key(ctx, table), id, data).Err()
}

func GetOne(ctx context.Context, id string, obj proto.Message) (err error) {
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())

	client := GetClient()
	ret := client.HGet(ctx, Key(ctx, table), id)
	if ret.Err() != nil {
		return ret.Err()
	}
	data, err := ret.Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, obj)
}

func DelOne(ctx context.Context, id string, obj proto.Message) (err error) {
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())

	client := GetClient()
	ret := client.HDel(ctx, Key(ctx, table), id)
	return ret.Err()
}

func GetAll(ctx context.Context, ids []string, results any) ([]string, error) {
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

	elementType := sliceVal.Type().Elem()
	newElem := reflect.New(elementType)

	obj := newElem.Addr().Interface().(proto.Message)
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())
	client := GetClient()
	ret := client.HMGet(ctx, Key(ctx, table), ids...)
	if ret.Err() != nil {
		return nil, ret.Err()
	}
	list := []string{}
	for i, val := range ret.Val() {
		if val == nil {
			list = append(list, ids[i])
			// sliceVal = reflect.Append(sliceVal, reflect.ValueOf(nil))
		} else {
			msg := proto.Clone(obj)
			json.Unmarshal([]byte(val.(string)), msg)
			sliceVal = reflect.Append(sliceVal, reflect.ValueOf(msg))
		}
		sliceVal = sliceVal.Slice(0, sliceVal.Cap())
	}
	resultsVal.Elem().Set(sliceVal.Slice(0, sliceVal.Cap()))
	return list, nil
}
