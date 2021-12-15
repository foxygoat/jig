function(input) {
  stream: [
    { response: 'Hello ' + input.request.message },
    { response: 'Goodbye ' + input.request.message },
  ],
}
