// greet.Greeter.HelloServerStream (Server streaming)

// Input:
// {
//   request: {  // HelloRequest
//     firstName: '',  // string
//   },
// }

function HelloServerStream(input, metadata) {
  return {
    stream: [
      {  // HelloResponse
        greeting: '',  // string
      },
    ],
  }
}
