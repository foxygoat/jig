function(input)
  if input.request.message == 'Bart' then
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
        message: 'eat my shorts',
        details: [
          {
            '@type': 'type.googleapis.com/google.protobuf.Duration',
            value: '101.212s',
          },
        ],
      },
    }
  else
    {
      response: {
        response: 'Hello ' + input.request.message,
      },
    }
