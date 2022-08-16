// greet.Greeter.HelloClientStream (Client streaming)

// Input:
// {
//   stream: [
//     {  // HelloRequest
//       firstName: "",  // string
//     },
//   ],
// }

function HelloClientStream(input, metadata) {
  return {
    response: {  // HelloResponse
      greeting: "",  // string
    },
  }
}
