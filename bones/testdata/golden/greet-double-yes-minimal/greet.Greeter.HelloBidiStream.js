// greet.Greeter.HelloBidiStream (Bidirectional streaming)

// Input:
// {
//   request: {  // greet.HelloRequest
//   },
// }

function HelloBidiStream(input) {
  return {
    stream: [
      {  // greet.HelloResponse
      },
    ],
  }
}
