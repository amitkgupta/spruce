package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

type Injector struct {
	root map[interface{}]interface{}
	ttl  int
}

func (i Injector) resolve(node string, args string) (interface{}, error) {
	DEBUG("%s: injection detected: (( inject %s ))", node, args)
	re := regexp.MustCompile(`\s+`)
	targets := re.Split(strings.Trim(args, " \t\r\n"), -1)

	val := []interface{}{}
	for _, target := range targets {
		v, err := i.resolveKey(target)
		if err != nil {
			return nil, err
		}
		if v != nil && reflect.TypeOf(v).Kind() != reflect.Map {
			return nil, fmt.Errorf("target %s is type `%s` not `map`, cannot inject its keys", target, reflect.TypeOf(v).Kind())
		}
		val = append(val, v)
	}
	return val, nil
}

func inject(parent interface{}, child interface{}, key interface{}) error {
	if parent == nil || reflect.TypeOf(parent).Kind() != reflect.Map {
		return fmt.Errorf("UNSUPPORTED FEATURE: injecting into things other than maps is currently unsupported")
	}
	if child == nil || reflect.TypeOf(child).Kind() != reflect.Slice {
		return fmt.Errorf("SPRUCE BUG DETECTED: Injector should return a []interface{} at all times and did not")
	}
	DEBUG("DELETING %s", key)
	delete(parent.(map[interface{}]interface{}), key)
	for _, e := range child.([]interface{}) {
		if e != nil && reflect.TypeOf(e).Kind() == reflect.Map {
			for k, v := range e.(map[interface{}]interface{}) {
				DEBUG("  -> injecting `%#v` at `%s`", v, k)
				var injection interface{}
				if v != nil && reflect.TypeOf(v).Kind() == reflect.Map {
					injection = make(map[interface{}]interface{})
					deepCopy(injection, v)
				} else {
					injection = v
				}
				parent.(map[interface{}]interface{})[k] = injection
			}
		} else {
			return fmt.Errorf("SPRUCE BUG DETECTED: Injector should validate values are maps, and let one by")
		}
	}
	return nil
}

func (i Injector) recurse(parent interface{}, key string, value interface{}) (interface{}, error) {
	if should, args := parseInjectOp(value); should {
		if i.ttl -= 1; i.ttl <= 0 {
			return nil, fmt.Errorf("possible recursion detected in call to (( inject ))")
		}
		value, err := i.resolve(fmt.Sprintf("$.%s", key), args)
		if err != nil {
			return nil, err
		}
		i.ttl += 1
		err = inject(parent, value, key)
		if err != nil {
			return nil, err
		}
	}
	return parent, nil
}

func (i Injector) resolveKey(key string) (interface{}, error) {
	DEBUG("  -> resolving reference to `%s`", key)
	val, err := resolveNode(key, i.root)
	if err != nil {
		return nil, fmt.Errorf("Unable to resolve `%s`: `%s", key, err)
	}

	if val != nil && reflect.TypeOf(val).Kind() == reflect.Map {
		for k, v := range val.(map[interface{}]interface{}) {
			//$			FIXME: some kind of iteration-order-based bug resulting in not always injecting all the things
			// Maybe it's not getting an updated copy of val after recursion?
			path := fmt.Sprintf("%s.%s", key, k)
			DEBUG("RECURSION START: %s", path)
			val, err = i.recurse(val, path, v)
			DEBUG("RECURSION END: %s", path)
			if err != nil {
				return nil, err
			}
		}
	} else if val != nil && reflect.TypeOf(val).Kind() == reflect.Slice {
		for j, e := range val.([]interface{}) {
			val, err = i.recurse(val, fmt.Sprintf("%s.[%d]", key, j), e)
			if err != nil {
				return nil, err
			}
		}
	}
	return val, nil
}

func parseInjectOp(o interface{}) (bool, string) {
	if o != nil && reflect.TypeOf(o).Kind() == reflect.String {
		re := regexp.MustCompile(`^\Q((\E\s*inject\s+(.+)\s*\Q))\E$`)
		if re.MatchString(o.(string)) {
			keys := re.FindStringSubmatch(o.(string))
			return true, keys[1]
		}
	}
	return false, ""
}

func (i Injector) PostProcess(o interface{}, node string) (interface{}, string, error) {
	if should, args := parseInjectOp(o); should {
		i.ttl = 64
		val, err := i.resolve(node, args)
		if err != nil {
			return nil, "error", fmt.Errorf("%s: %s", node, err.Error())
		}
		return val, "inject", nil
	}
	return nil, "ignore", nil
}
