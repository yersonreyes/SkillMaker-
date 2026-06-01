export const environment = {
  production: false,
  apiBaseUrl: 'http://localhost:3000/api',
  googleClientId: '600011279090-fu6th45fnkk8897he1cgn89flruhe5ml.apps.googleusercontent.com',
  // En dev con Gmail personal queda vacio para que el popup acepte la cuenta.
  // En prod con Workspace se setea con el dominio corporativo (RT-13).
  googleHostedDomain: '',
};
