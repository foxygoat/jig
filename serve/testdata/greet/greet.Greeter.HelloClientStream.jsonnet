function(input, metadata) {
  response: {
    local names = [req.firstName for req in input.stream],
    greeting: '💃 jig [client]: Hello ' + std.join(' and ', names),
  },
  header: {
    count: [std.toString(std.length(input.stream))],
  },
  trailer: {
    size: [std.toString(std.length($.response.greeting))],
  },
}
