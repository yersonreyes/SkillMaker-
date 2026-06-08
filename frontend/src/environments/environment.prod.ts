export const environment = {
  production: true,
  apiBaseUrl: '/api',
  googleClientId: '600011279090-fu6th45fnkk8897he1cgn89flruhe5ml.apps.googleusercontent.com',
  // Vacio: el client OAuth es de Gmail personal (sin hosted domain corporativo).
  // Forzar un 'hd' aqui rechazaria el login. Setear solo con Workspace real.
  googleHostedDomain: '',
};
