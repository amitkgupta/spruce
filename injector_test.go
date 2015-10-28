package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestInjectorPostProcess(t *testing.T) {
	Convey("injector.PostProcess()", t, func() {
		injector := Injector{root: map[interface{}]interface{}{
			"inject_from_map": map[interface{}]interface{}{
				"new_key": 1,
			},
			"inject_from_array": []interface{}{
				map[interface{}]interface{}{
					"new_key": 2,
				},
			},
		}}
		Convey("returns nil, \"ignore\", nil", func() {
			Convey("when given anything other than a string", func() {
				val, action, err := injector.PostProcess(12345, "nodepath")
				So(val, ShouldBeNil)
				So(err, ShouldBeNil)
				So(action, ShouldEqual, "ignore")
			})
			Convey("when given a '(( grab ))' string", func() {
				val, action, err := injector.PostProcess("(( grab ))", "nodepath")
				So(val, ShouldBeNil)
				So(err, ShouldBeNil)
				So(action, ShouldEqual, "ignore")
			})
			Convey("when given a non-'(( inject .* ))' string", func() {
				val, action, err := injector.PostProcess("regular old string", "nodepath")
				So(val, ShouldBeNil)
				So(err, ShouldBeNil)
				So(action, ShouldEqual, "ignore")
			})
			Convey("when given a quoted-'(( inject .* ))' string", func() {
				val, action, err := injector.PostProcess("\"(( inject inject_from_map ))\"", "nodepath")
				So(val, ShouldBeNil)
				So(err, ShouldBeNil)
				So(action, ShouldEqual, "ignore")
			})
		})
		Convey("Returns an error if unable to resolve the node to inject from", func() {
			val, action, err := injector.PostProcess("(( inject inject_from_here ))", "nodepath")
			So(val, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldStartWith, "nodepath: Unable to resolve `inject_from_here`:")
			So(action, ShouldEqual, "error")
		})
		Convey("Returns value, \"inject\", nil on successful resolve of map node", func() {
			val, action, err := injector.PostProcess("(( inject inject_from_map ))", "nodepath")
			So(val, ShouldResemble, []interface{}{map[interface{}]interface{}{
				"new_key": 1,
			}})
			So(err, ShouldBeNil)
			So(action, ShouldEqual, "inject")
		})
		Convey("Returns value, \"inject\", nil on successful resolve of an array node", func() {
			val, action, err := injector.PostProcess("(( inject inject_from_array.[0] ))", "nodepath")
			So(val, ShouldResemble, []interface{}{map[interface{}]interface{}{
				"new_key": 2,
			}})
			So(err, ShouldBeNil)
			So(action, ShouldEqual, "inject")
		})
		Convey("Handles multiple inject requests inline by returning an array", func() {
			val, action, err := injector.PostProcess("(( inject inject_from_map inject_from_array.[0] ))", "nodepath")
			So(val, ShouldResemble, []interface{}{
				map[interface{}]interface{}{
					"new_key": 1,
				},
				map[interface{}]interface{}{
					"new_key": 2,
				},
			})
			So(err, ShouldBeNil)
			So(action, ShouldEqual, "inject")
		})
		Convey("Errors on the first problem of a multiple reference request", func() {
			val, action, err := injector.PostProcess("(( inject inject_from_map undefined.val othervalue.to.find ))", "nodepath")
			So(val, ShouldBeNil)
			So(action, ShouldEqual, "error")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "nodepath: Unable to resolve `undefined.val`: `undefined` could not be found in the YAML datastructure")
		})
		Convey("Extra whitespace is ok", func() {
			val, action, err := injector.PostProcess("((	  inject inject_from_map		inject_from_array.[0]     ))", "nodepath")
			So(val, ShouldResemble, []interface{}{
				map[interface{}]interface{}{
					"new_key": 1,
				},
				map[interface{}]interface{}{
					"new_key": 2,
				},
			})
			So(err, ShouldBeNil)
			So(action, ShouldEqual, "inject")
		})
		Convey("Errors when you try to reference a value that is not a map", func() {
			Convey("When referencing an array", func() {
				val, action, err := injector.PostProcess("(( inject inject_from_array ))", "nodepath")
				So(val, ShouldBeNil)
				So(action, ShouldEqual, "error")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "nodepath: target inject_from_array is type `slice` not `map`, cannot inject its keys")
			})
			Convey("When referencing a scaar", func() {
				val, action, err := injector.PostProcess("(( inject inject_from_map.new_key ))", "nodepath")
				So(val, ShouldBeNil)
				So(action, ShouldEqual, "error")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "nodepath: target inject_from_map.new_key is type `int` not `map`, cannot inject its keys")

			})
			Convey("Even if only one of multiple values referenced are bad", func() {
				val, action, err := injector.PostProcess("(( inject inject_from_map inject_from_array.[0] inject_from_array ))", "nodepath")
				So(val, ShouldBeNil)
				So(action, ShouldEqual, "error")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "nodepath: target inject_from_array is type `slice` not `map`, cannot inject its keys")
			})
		})
	})
}
