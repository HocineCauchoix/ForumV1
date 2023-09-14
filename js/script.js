
  
  /* Code JS pour déclencher les animations */
  
  // Animation slide-in-bottom pour le formulaire
  const formWrapper = document.querySelector('.form-wrapper');
  formWrapper.classList.add('slide-in-bottom');
  
  // Animation fade-in pour les champs de formulaire
  const inputs = document.querySelectorAll('input[type="text"], input[type="email"], input[type="password"]');
  inputs.forEach(input => {
    input.classList.add('fade-in');
  });
  