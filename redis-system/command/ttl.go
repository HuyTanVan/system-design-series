package command

import (
	"redis-clone/resp"
	"strconv"
	"time"
)

// note: if we do (return resp.Value{}), this will keep the cli hanging on
// get remaining time (seconds) to live of a key
// eg: TTL key1
func (d *Dispatcher) ttl(args []resp.Value) resp.Value {
	if len(args) != 1 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for TTL"}
	}
	remaining := d.store.TTL(args[0].Bulk)
	if remaining == -1 {
		return resp.Value{Typ: "error", Str: "ERR key exists but has no associated expire"}
	}
	if remaining == -2 {
		return resp.Value{Typ: "error", Str: "ERR key has been expired"}
	}
	// Implementation for TTL command
	return resp.Value{Typ: "integer", Num: remaining} // -1 means key exists but has no associated expire, -2 means key does not exist
}

// EXPIRE key seconds
func (d *Dispatcher) expire(args []resp.Value) resp.Value {
	if len(args) != 2 {
		return resp.Value{Typ: "error", Str: "ERR wrong number of arguments for EXPIRE"}
	}
	secs, _ := strconv.Atoi(args[1].Bulk)
	if secs <= 0 {
		return resp.Value{Typ: "error", Str: "ERR invalid expire time"}
	}

	result := d.store.SetExpiry(args[0].Bulk, time.Duration(secs)*time.Second)

	if result == -2 {
		return resp.Value{Typ: "integer", Num: result} // -2 = key doesn't exist
	}
	if result == -1 {
		return resp.Value{Typ: "integer", Num: result} // -1 = key exists but no associated expire
	}
	return resp.Value{Typ: "integer", Num: result} // 1 = success
}
