function(input) {
  stream: [
    { greeting: '💃 jig [server]: Hello ' + input.request.firstName },
    { greeting: '💃 jig [server]: Goodbye ' + input.request.firstName },
  ],
}
