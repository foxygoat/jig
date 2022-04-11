// httpgreet.HttpGreeter.PostHello (Unary)

// Input:
// {
//   request: {
//     firstName: "",  // string
//     lastName: "",  // string
//   },
// }

function(input) {
  response: {
    greeting: "Thanks for the post, " + input.request.firstName,  // string
  },
}
