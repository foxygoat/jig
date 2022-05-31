// greet.Greeter.HelloServerStream (Server streaming)

// Input:
// {
//   request: {  // HelloRequest
//     firstName: '',  // string
//   },
// }

function HelloServerStream(input) {
  return {
    stream: [
      {  // HelloResponse
        greeting: '',  // string
      },
    ],
  }
}
