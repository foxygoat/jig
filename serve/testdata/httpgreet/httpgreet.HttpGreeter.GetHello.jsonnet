// httpgreet.HttpGreeter.GetHello (Unary)

// Input:
// {
//   request: {
//     firstName: "",  // string
//     lastName: "",  // string
//   },
// }

function(input) {
  response: {
    greeting: "httpgreet: Hello, " + input.request.firstName,  // string
  },
}
