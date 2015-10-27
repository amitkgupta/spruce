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
	DEBUG("%s: resolving (( inject %s ))", node, args)
	re := regexp.MustCompile(`\s+`)
	targets := re.Split(strings.Trim(args, " \t\r\n"), -1)

	if len(targets) <= 1 {
		val, err := i.resolveKey(targets[0])
		return val, err
	}

	val := []interface{}{}
	for _, target := range targets {
		v, err := i.resolveKey(target)
		if err != nil {
			return nil, err
		}
		if v != nil && reflect.TypeOf(v).Kind() == reflect.Slice {
			for j := 0; j < reflect.ValueOf(v).Len(); j++ {
				val = append(val, reflect.ValueOf(v).Index(j).Interface())
			}
		} else {
			val = append(val, v)
		}
	}
	return val, nil
}

func (i Injector) resolveKey(key string) (interface{}, error) {
	DEBUG("  -> resolving reference to `%s`", key)
	val, err := resolveNode(key, i.root)
	if err != nil {
		return nil, fmt.Errorf("Unable to resolve `%s`: `%s", key, err)
	}

	if should, args := parseGrabOp(val); should {
		if i.ttl -= 1; i.ttl <= 0 {
			return "", fmt.Errorf("possible recursion detected in call to (( inject ))")
		}
		val, err = i.resolve(key, args)
		i.ttl += 1
		return val, err
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
		DEBUG("%s: injecting keys from %#v", node, val)
		return val, "inject", nil
	}
	return nil, "ignore", nil
}
