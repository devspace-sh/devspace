import React from 'react';

function Home() {
  React.useEffect(() => {
    window.location.href = './quickstart';
  }, []);
  
  return null
}

export default Home;
