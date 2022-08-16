// greet.Greeter.HelloServerStream (Server streaming)

// Input:
// {
//   request: {  // HelloRequest
//     firstName: '',  // string
//   },
// }

function(input, metadata) {
  stream: [
    {  // HelloResponse
      greeting: '',  // string
    },
  ],
}
