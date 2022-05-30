// greet.Greeter.HelloClientStream (Client streaming)

// Input:
// {
//   stream: [
//     {  // HelloRequest
//       firstName: "",  // string
//     },
//   ],
// }

function HelloClientStream(input) {
  return {
    response: {  // HelloResponse
      greeting: "",  // string
    },
  }
}
