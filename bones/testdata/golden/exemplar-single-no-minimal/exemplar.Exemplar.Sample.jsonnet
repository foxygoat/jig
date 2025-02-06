// exemplar.Exemplar.Sample (Unary)

// Input:
// {
//   request: {  // SampleRequest
//     name: '',  // string
//   },
// }

function(input) {
  response: {  // SampleResponse
    aBool: false,  // bool
    aInt32: 0,  // int32
    aSint32: 0,  // sint32
    aSfixed32: 0,  // sfixed32
    aUint32: 0,  // uint32
    aFixed32: 0,  // fixed32
    aInt64: 0,  // int64
    aSint64: 0,  // sint64
    aSfixed64: 0,  // sfixed64
    aUint64: 0,  // uint64
    aFixed64: 0,  // fixed64
    aFloat: 0.0,  // float
    aDouble: 0.0,  // double
    aString: '',  // string
    aBytes: '',  // bytes
    aEnum: 'SAMPLE_ENUM_FIRST',  // SampleEnum
    aMessage: {  // SampleMessage1
      field: '',  // string
      repeat: [0],  // repeated int32
    },
    aMap: {  // map<int32, bool>
      "0": false,
    },
    aDeepMap: {  // map<string, SampleMessage2>
      "key": {
        weirdFieldName1: '',  // string
        aStringList: [''],  // repeated string
        aMsgList: [{}],  // repeated SampleMessage1 (see example above)
      },
    },
    aIntList: [0],  // repeated int32
    aEnumList: ['SAMPLE_ENUM_FIRST'],  // repeated SampleEnum
    aMessageList: [{}],  // repeated SampleMessage1 (see example above)
    // aStringOneof: '',  // string (one-of a_oneof)
    // aEnumOneof: 'SAMPLE_ENUM_FIRST',  // SampleEnum (one-of a_oneof)
    // aMessageOneof: {},  // SampleMessage1 (one-of a_oneof, see example above)
    recursive: {},  // SampleResponse (see example above)
  },
}
