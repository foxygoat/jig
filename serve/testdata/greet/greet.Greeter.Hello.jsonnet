function(input, metadata)
  if input.request.firstName == 'Bart' then
    {
      header: {
        eat: ['my', 'shorts'],
      },
      trailer: {
        dont: ['have'],
        a: ['cow'],
      },
      status: {
        code: 3,
        message: '💃 jig [unary]: eat my shorts',
        details: [
          {
            // A type dynamically loaded from a pb file (duration.pb), that is
            // not referenced in the main greeter.pb
            '@type': 'type.googleapis.com/google.protobuf.Duration',
            value: '42s',
          },
          {
            // A type with an extension field. MethodOptions and the http
            // extension field are both present in the main greeter.pb
            '@type': 'type.googleapis.com/google.protobuf.MethodOptions',
            '[google.api.http]': {
              post: '/api/greet/hello',
            },
          },
        ],
      },
    }
  else
    {
      response: {
        greeting: '💃 jig [unary]: Hello ' + input.request.firstName,
      },
    }
