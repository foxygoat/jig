// greet.Greeter.HelloServerStream (Server streaming)

// Input:
// {
//   request: {  // greet.HelloRequest
//   },
// }

function HelloServerStream(input) {
  return {
    stream: [
      {  // greet.HelloResponse
      },
    ],
  }
}
