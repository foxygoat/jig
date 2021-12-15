function(input) {
  local response =
    if input.request == null then
      []
    else
      [{ response: 'Hello ' + input.request.message }],
  stream: response,
}
