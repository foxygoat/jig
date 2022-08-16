// greet.Greeter.HelloClientStream (Client streaming)

// Input:
// {
//   stream: [
//     {  // greet.HelloRequest
//     },
//   ],
// }

function HelloClientStream(input, metadata) {
  return {
    response: {  // greet.HelloResponse
    },
  }
}
