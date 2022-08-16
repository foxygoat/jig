// greet.Greeter.HelloServerStream (Server streaming)

// Input:
// {
//   request: {  // greet.HelloRequest
//   },
// }

function HelloServerStream(input, metadata) {
  return {
    stream: [
      {  // greet.HelloResponse
      },
    ],
  }
}
