// greet.Greeter.HelloServerStream (Server streaming)

// Input:
// {
//   request: {
//     firstName: '',  // string
//   },
// }

function HelloServerStream(input) {
  return {
    stream: [
      {
        greeting: '',  // string
      },
    ],
  }
}
