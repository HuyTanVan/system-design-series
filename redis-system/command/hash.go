package command

import (
	"redis-clone/resp"
)

// HSET key field value [field value ...]
// Time: O(1) for each field set. O(N) for N fields.
func (d *Dispatcher) hset(args []resp.Value) resp.Value {
	if len(args) < 3 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for HSET"}
	}
	// eg: key1 field1 value1 field2 value2, len(args) = 5, (len(args)-1)%2 = 0, valid
	if (len(args)-1)%2 != 0 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for HSET"}
	}
	count := 0
	for i := 1; i < len(args); i += 2 {
		count += d.store.HSet(args[0].Bulk, args[i].Bulk, args[i+1].Bulk)
	}
	return resp.Value{Typ: "integer", Num: count}
}

// HGET key field
// Time: O(1)
func (d *Dispatcher) hget(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for HGET"}
	}
	val, ok := d.store.HGet(args[0].Bulk, args[1].Bulk)
	if !ok {
		return resp.Value{Typ: "null"} // null
	}
	return resp.Value{Typ: "bulk", Bulk: val}
	// return resp.Value{Typ: "bulk", Bulk: d.store.HGet(args[0].Bulk, args[1].Bulk)}
}

// HDEL key field [field ...]
// Time: O(1) for each field deletion. O(N) for N fields.
func (d *Dispatcher) hdel(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for HDEL"}
	}
	count := 0
	for i := 1; i < len(args); i++ {
		count += d.store.HDel(args[0].Bulk, args[i].Bulk)

	}
	return resp.Value{Typ: "integer", Num: count}
}

// HGETALL key
// Time: O(N) where N is the number of fields in the hash.
func (d *Dispatcher) hgetall(args []resp.Value) resp.Value {
	if len(args) < 1 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for HGETALL"}
	}
	fields := d.store.HGetAll(args[0].Bulk)
	result := make([]resp.Value, 0, len(fields)*2)
	for field, value := range fields {
		result = append(result, resp.Value{Typ: "bulk", Bulk: field})
		result = append(result, resp.Value{Typ: "bulk", Bulk: value})
	}
	return resp.Value{Typ: "array", Array: result}
}

// HEXISTS key field
// Time: O(1)
func (d *Dispatcher) hexists(args []resp.Value) resp.Value {
	if len(args) < 2 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for HEXISTS"}
	}
	// 1 = yes, 0 = no
	exists := d.store.HExists(args[0].Bulk, args[1].Bulk)
	return resp.Value{Typ: "integer", Num: exists}
}

// HLEN key
func (d *Dispatcher) hlen(args []resp.Value) resp.Value {
	if len(args) < 1 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for HLEN"}
	}
	length := d.store.Hlen(args[0].Bulk)
	return resp.Value{Typ: "integer", Num: length}
}
