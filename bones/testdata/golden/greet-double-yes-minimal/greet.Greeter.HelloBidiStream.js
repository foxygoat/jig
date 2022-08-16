// greet.Greeter.HelloBidiStream (Bidirectional streaming)

// Input:
// {
//   request: {  // greet.HelloRequest
//   },
// }

function HelloBidiStream(input, metadata) {
  return {
    stream: [
      {  // greet.HelloResponse
      },
    ],
  }
}
