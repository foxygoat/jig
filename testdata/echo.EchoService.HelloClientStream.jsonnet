function(input) {
  response: {
    local messages = [req.message for req in input.stream],
    response: 'Hello ' + std.join(' and ', messages),
  },
  header: {
    count: [std.toString(std.length(input.stream))],
  },
  trailer: {
    size: [std.toString(std.length($.response.response))],
  },
}
