function(input)
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
            '@type': 'type.googleapis.com/google.protobuf.Duration',
            value: '42s',
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
