// Code generated by "jade.go"; DO NOT EDIT.

package jade

import (
	pool "github.com/valyala/bytebufferpool"
)

func tpl_case(buffer *pool.ByteBuffer) {

	var friends1 = 10
	switch friends1 {
	case 0:
		buffer.WriteString(`<p>you have no friends1</p>`)

	case 1:
		buffer.WriteString(`<p>you have a friend</p>`)

	default:
		buffer.WriteString(`<p>you have `)
		WriteInt(int64(friends1), buffer)
		buffer.WriteString(` friends1</p>`)

	}
	var friends2 = 0
	switch friends2 {
	case 0:
		fallthrough
	case 1:
		buffer.WriteString(`<p>you have very few friends2</p>`)

	default:
		buffer.WriteString(`<p>you have `)
		WriteInt(int64(friends2), buffer)
		buffer.WriteString(` friends2</p>`)

	}
	var friends3 = 0
	switch friends3 {
	case 0:
		break
	case 1:
		buffer.WriteString(`<p>you have very few friends3</p>`)

	default:
		buffer.WriteString(`<p>you have `)
		WriteInt(int64(friends3), buffer)
		buffer.WriteString(` friends3</p>`)

	}
	var friends = 1
	switch friends {
	case 0:
		buffer.WriteString(`<p>you have no friends</p>`)

	case 1:
		buffer.WriteString(`<p>you have a friend</p>`)

	default:
		buffer.WriteString(`<p>you have `)
		WriteInt(int64(friends), buffer)
		buffer.WriteString(` friends</p>`)

	}

}