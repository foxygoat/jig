function(input)
  if input.request != null && input.request.firstName == 'Bart' then
    {
      status: {
        code: 3,  // InvalidArgument
        message: '💃 jig [bidi]: eat my shorts',
      },
      // Without this header, the content-type is sent in the trailer
      // as there is nothing in the body. This is a "trailer-only" response.
      header: {
        eat: ['his', 'shorts'],
      },
    }
  else
    {
      local response =
        if input.request == null then
          []
        else
          [{ greeting: '💃 jig [bidi]: Hello ' + input.request.firstName }],
      stream: response,
    }
