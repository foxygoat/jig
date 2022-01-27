// exemplar.Exemplar.WellKnown (Unary)

// Input:
// {
//   request: {
//     name: "",  // string
//   },
// }

function WellKnown(input) {
  return {
    response: {
      any: {  // Any
        "@type": "type.googleapis.com/google.protobuf.Duration",
        value: "0s",
      },
      api: {  // Api
        name: "",  // string
        methods: [  // repeated Method
          {
            name: "",  // string
            requestTypeUrl: "",  // string
            requestStreaming: false,  // bool
            responseTypeUrl: "",  // string
            responseStreaming: false,  // bool
            options: [  // repeated Option
              {
                name: "",  // string
                value: {  // Any
                  "@type": "type.googleapis.com/google.protobuf.Duration",
                  value: "0s",
                },
              }
            ],
            syntax: "SYNTAX_PROTO3",  // Syntax
          }
        ],
        options: [  // repeated Option
          {
            name: "",  // string
            value: {  // Any
              "@type": "type.googleapis.com/google.protobuf.Duration",
              value: "0s",
            },
          }
        ],
        version: "",  // string
        sourceContext: {  // SourceContext
          fileName: "",  // string
        },
        mixins: [  // repeated Mixin
          {
            name: "",  // string
            root: "",  // string
          }
        ],
        syntax: "SYNTAX_PROTO3",  // Syntax
      },
      boolValue: false,  // BoolValue
      bytesValue: "",  // BytesValue
      doubleValue: 0.0,  // DoubleValue
      duration: "0s",  // Duration
      empty: {},  // Empty
      anEnum: {  // Enum
        name: "",  // string
        enumvalue: [  // repeated EnumValue
          {
            name: "",  // string
            number: 0,  // int32
            options: [  // repeated Option
              {
                name: "",  // string
                value: {  // Any
                  "@type": "type.googleapis.com/google.protobuf.Duration",
                  value: "0s",
                },
              }
            ],
          }
        ],
        options: [  // repeated Option
          {
            name: "",  // string
            value: {  // Any
              "@type": "type.googleapis.com/google.protobuf.Duration",
              value: "0s",
            },
          }
        ],
        sourceContext: {  // SourceContext
          fileName: "",  // string
        },
        syntax: "SYNTAX_PROTO3",  // Syntax
      },
      enumValue: {  // EnumValue
        name: "",  // string
        number: 0,  // int32
        options: [  // repeated Option
          {
            name: "",  // string
            value: {  // Any
              "@type": "type.googleapis.com/google.protobuf.Duration",
              value: "0s",
            },
          }
        ],
      },
      field: {  // Field
        kind: "TYPE_DOUBLE",  // Kind
        cardinality: "CARDINALITY_OPTIONAL",  // Cardinality
        number: 0,  // int32
        name: "",  // string
        typeUrl: "",  // string
        oneofIndex: 0,  // int32
        packed: false,  // bool
        options: [  // repeated Option
          {
            name: "",  // string
            value: {  // Any
              "@type": "type.googleapis.com/google.protobuf.Duration",
              value: "0s",
            },
          }
        ],
        jsonName: "",  // string
        defaultValue: "",  // string
      },
      fieldMask: "field1.field2,field3",  // FieldMask
      floatValue: 0.0,  // FloatValue
      int32Value: 0,  // Int32Value
      int64Value: 0,  // Int64Value
      listValue: ["https://developers.google.com/protocol-buffers/docs/reference/google.protobuf#value"],  // ListValue
      method: {  // Method
        name: "",  // string
        requestTypeUrl: "",  // string
        requestStreaming: false,  // bool
        responseTypeUrl: "",  // string
        responseStreaming: false,  // bool
        options: [  // repeated Option
          {
            name: "",  // string
            value: {  // Any
              "@type": "type.googleapis.com/google.protobuf.Duration",
              value: "0s",
            },
          }
        ],
        syntax: "SYNTAX_PROTO3",  // Syntax
      },
      mixin: {  // Mixin
        name: "",  // string
        root: "",  // string
      },
      nullValue: null,  // NullValue
      anOption: {  // Option
        name: "",  // string
        value: {  // Any
          "@type": "type.googleapis.com/google.protobuf.Duration",
          value: "0s",
        },
      },
      sourceContext: {  // SourceContext
        fileName: "",  // string
      },
      stringValue: "",  // StringValue
      struct: {  // Struct
        structField: "https://developers.google.com/protocol-buffers/docs/reference/google.protobuf#value",
      },
      timestamp: "2006-01-02T15:04:05.999999999Z",  // Timestamp
      type: {  // Type
        name: "",  // string
        fields: [  // repeated Field
          {
            kind: "TYPE_DOUBLE",  // Kind
            cardinality: "CARDINALITY_OPTIONAL",  // Cardinality
            number: 0,  // int32
            name: "",  // string
            typeUrl: "",  // string
            oneofIndex: 0,  // int32
            packed: false,  // bool
            options: [  // repeated Option
              {
                name: "",  // string
                value: {  // Any
                  "@type": "type.googleapis.com/google.protobuf.Duration",
                  value: "0s",
                },
              }
            ],
            jsonName: "",  // string
            defaultValue: "",  // string
          }
        ],
        oneofs: [""],  // repeated string
        options: [  // repeated Option
          {
            name: "",  // string
            value: {  // Any
              "@type": "type.googleapis.com/google.protobuf.Duration",
              value: "0s",
            },
          }
        ],
        sourceContext: {  // SourceContext
          fileName: "",  // string
        },
        syntax: "SYNTAX_PROTO3",  // Syntax
      },
      uint32Value: 0,  // UInt32Value
      uint64Value: 0,  // UInt64Value
      value: "https://developers.google.com/protocol-buffers/docs/reference/google.protobuf#value",  // Value
    },
  }
}
