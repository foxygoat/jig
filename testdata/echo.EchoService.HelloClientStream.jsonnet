function(input) {
  response: {
    local messages = [req.message for req in input.stream],
    response: 'Hello ' + std.join(' and ', messages),
  },
}
