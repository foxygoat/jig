function(input)
  local isBart = input.request.message == 'Bart';
  {
    [if !isBart then 'response']: {
      response: 'Hello ' + input.request.message,
    },

    [if isBart then 'status']: {
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
