// greet.Greeter.HelloClientStream (Client streaming)

// Input:
// {
//   stream: [
//     {  // HelloRequest
//       firstName: "",  // string
//     },
//   ],
// }

function(input, metadata) {
  response: {  // HelloResponse
    greeting: "",  // string
  },
}
