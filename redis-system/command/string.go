package command

import (
	"redis-clone/resp"
)

// Command handlers for string operations (SET, GET, DEL)

// PING [message]
func (d *Dispatcher) ping(args []resp.Value) resp.Value {
	if len(args) == 0 {
		return resp.Value{Typ: "string", Str: "PONG"}
	}
	return resp.Value{Typ: "bulk", Bulk: args[0].Bulk}
}

// redis command: SET key value.
// Time Complexity: O(1), this operation only allows one k-v at a time
func (d *Dispatcher) set(args []resp.Value) resp.Value {
	if len(args) < 1 {
		return resp.Value{Typ: "error", Str: "WAITING FOR MORE ARGUMENTS"}
	}
	if len(args) < 2 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for SET"}
	}
	// fmt.Println("KEY:", args[0].Bulk, "VALUE:", args[1].Bulk)
	d.store.Set(args[0].Bulk, args[1].Bulk)
	return resp.Value{Typ: "string", Str: "OK"}
}

// Time Complexity: O(1)
func (d *Dispatcher) get(args []resp.Value) resp.Value {
	val, ok := d.store.Get(args[0].Bulk)
	if !ok {
		return resp.Value{Typ: "null"} // null
	}
	return resp.Value{Typ: "bulk", Bulk: val}
}

// Time Complexity: O(N) where N is the number of keys to delete. O(1) for each key deletion.
func (d *Dispatcher) del(args []resp.Value) resp.Value {
	if len(args) < 1 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for DEL"}
	}
	count := 0
	for _, arg := range args {
		count += d.store.Del(arg.Bulk)
	}
	return resp.Value{Typ: "integer", Num: count}
}
