// greet.Greeter.HelloClientStream (Client streaming)

// Input:
// {
//   stream: [
//     {
//       firstName: "",  // string
//     },
//   ],
// }

function HelloClientStream(input) {
  return {
    response: {
      greeting: "",  // string
    },
  }
}
