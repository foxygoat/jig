// greet.Greeter.HelloBidiStream (Bidirectional streaming)

// Input:
// {
//   request: {  // HelloRequest
//     firstName: "",  // string
//   },
// }

function(input, metadata) {
  stream: [
    {  // HelloResponse
      greeting: "",  // string
    },
  ],
}
