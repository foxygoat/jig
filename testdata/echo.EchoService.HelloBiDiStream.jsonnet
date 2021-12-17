function(input)
  if input.request != null && input.request.message == 'Bart' then
    {
      status: {
        code: 3,
        message: 'eat my shorts',
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
          [{ response: 'Hello ' + input.request.message }],
      stream: response,
    }
