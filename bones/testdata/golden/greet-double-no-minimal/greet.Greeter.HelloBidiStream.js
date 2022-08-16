// greet.Greeter.HelloBidiStream (Bidirectional streaming)

// Input:
// {
//   request: {  // HelloRequest
//     firstName: "",  // string
//   },
// }

function HelloBidiStream(input, metadata) {
  return {
    stream: [
      {  // HelloResponse
        greeting: "",  // string
      },
    ],
  }
}
