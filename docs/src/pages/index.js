import React from 'react';

function Home() {
  React.useEffect(() => {
    window.location.href = './getting-started/introduction';
  }, []);
  
  return null
}

export default Home;
