// greet.Greeter.HelloServerStream (Server streaming)

// Input:
// {
//   request: {  // HelloRequest
//     firstName: '',  // string
//   },
// }

function(input) {
  stream: [
    {  // HelloResponse
      greeting: '',  // string
    },
  ],
}
