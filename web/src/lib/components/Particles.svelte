<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  let canvas: HTMLCanvasElement;
  let ctx: CanvasRenderingContext2D;
  let animFrame: number;
  let mouse = { x: -9999, y: -9999 };

  interface Particle {
    x: number;
    y: number;
    vx: number;
    vy: number;
    size: number;
    opacity: number;
    hue: number;
  }

  const COUNT = 55;
  const CONNECT_DIST = 130;
  let particles: Particle[] = [];
  let W = 0;
  let H = 0;

  function init(): void {
    W = canvas.width = window.innerWidth;
    H = canvas.height = window.innerHeight;
    particles = Array.from({ length: COUNT }, () => ({
      x: Math.random() * W,
      y: Math.random() * H,
      vx: (Math.random() - 0.5) * 0.25,
      vy: (Math.random() - 0.5) * 0.25,
      size: Math.random() * 1.8 + 0.4,
      opacity: Math.random() * 0.25 + 0.08,
      hue: 250 + Math.random() * 30, // violet range
    }));
  }

  function draw(): void {
    ctx.clearRect(0, 0, W, H);

    // Connection lines
    for (let i = 0; i < particles.length; i++) {
      for (let j = i + 1; j < particles.length; j++) {
        const dx = particles[i].x - particles[j].x;
        const dy = particles[i].y - particles[j].y;
        const dist = Math.sqrt(dx * dx + dy * dy);
        if (dist < CONNECT_DIST) {
          const alpha = (1 - dist / CONNECT_DIST) * 0.06;
          ctx.strokeStyle = `hsla(265, 70%, 65%, ${alpha})`;
          ctx.lineWidth = 0.5;
          ctx.beginPath();
          ctx.moveTo(particles[i].x, particles[i].y);
          ctx.lineTo(particles[j].x, particles[j].y);
          ctx.stroke();
        }
      }
    }

    // Update & draw particles
    for (const p of particles) {
      // Mouse repel parallax
      const dx = mouse.x - p.x;
      const dy = mouse.y - p.y;
      const dist = Math.sqrt(dx * dx + dy * dy);
      if (dist < 180 && dist > 0) {
        const force = ((180 - dist) / 180) * 0.015;
        p.vx -= (dx / dist) * force;
        p.vy -= (dy / dist) * force;
      }

      p.x += p.vx;
      p.y += p.vy;

      // Wrap edges with padding
      if (p.x < -10) p.x = W + 10;
      if (p.x > W + 10) p.x = -10;
      if (p.y < -10) p.y = H + 10;
      if (p.y > H + 10) p.y = -10;

      // Gentle damping
      p.vx *= 0.998;
      p.vy *= 0.998;

      // Re-add tiny random drift
      p.vx += (Math.random() - 0.5) * 0.005;
      p.vy += (Math.random() - 0.5) * 0.005;

      // Draw
      ctx.fillStyle = `hsla(${p.hue}, 70%, 70%, ${p.opacity})`;
      ctx.beginPath();
      ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
      ctx.fill();
    }

    animFrame = requestAnimationFrame(draw);
  }

  function handleResize(): void {
    W = canvas.width = window.innerWidth;
    H = canvas.height = window.innerHeight;
  }

  function handleMouse(e: MouseEvent): void {
    mouse.x = e.clientX;
    mouse.y = e.clientY;
  }

  onMount(() => {
    ctx = canvas.getContext('2d')!;
    init();
    draw();
    window.addEventListener('resize', handleResize);
    window.addEventListener('mousemove', handleMouse);
  });

  onDestroy(() => {
    cancelAnimationFrame(animFrame);
    window.removeEventListener('resize', handleResize);
    window.removeEventListener('mousemove', handleMouse);
  });
</script>

<canvas bind:this={canvas} class="particles-canvas" aria-hidden="true"></canvas>

<style>
  .particles-canvas {
    position: fixed;
    inset: 0;
    z-index: 0;
    pointer-events: none;
  }
</style>
