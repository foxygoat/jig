// greet.Greeter.Hello (Unary)

// Input:
// {
//   request: {  // HelloRequest
//     firstName: "",  // string
//   },
// }

function Hello(input, metadata) {
  return {
    response: {  // HelloResponse
      greeting: "",  // string
    },
  }
}
