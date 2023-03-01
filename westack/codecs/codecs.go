package codecs

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// objectIDCodec is codec for Protobuf ObjectId
type objectIDCodec struct {
}

var objectIDPrimitiveType = reflect.TypeOf(primitive.ObjectID{})

// EncodeValue encodes primitive.ObjectID value to BSON value
func (e *objectIDCodec) EncodeValue(ectx bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	v := val.Interface().(primitive.ObjectID)
	enc, err := ectx.LookupEncoder(objectIDPrimitiveType)
	if err != nil {
		return err
	}
	return enc.EncodeValue(ectx, vw, reflect.ValueOf(v))
}

// DecodeValue decodes BSON value to primitive.ObjectID value
func (e *objectIDCodec) DecodeValue(ectx bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	enc, err := ectx.LookupDecoder(objectIDPrimitiveType)
	if err != nil {
		return err
	}
	var id primitive.ObjectID
	if err = enc.DecodeValue(ectx, vr, reflect.ValueOf(&id).Elem()); err != nil {
		return err
	}
	val.Set(reflect.ValueOf(id))
	return nil
}
