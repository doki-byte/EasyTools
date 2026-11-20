<template>
  <div class="game-selector">
    <div class="game-selector-header">
      <h2>🎮 小游戏</h2>
      <p class="subtitle">放松一下，玩个小游戏吧！</p>
    </div>
    <div class="game-list">
      <div class="game-card" @click="selectGame('2048')">
        <div class="game-icon">2048</div>
        <div class="game-name">2048</div>
        <div class="game-desc">滑动合并数字，达到2048！</div>
      </div>
      <div class="game-card" @click="selectGame('minesweeper')">
        <div class="game-icon">💣</div>
        <div class="game-name">扫雷</div>
        <div class="game-desc">找出所有地雷，挑战你的逻辑！</div>
      </div>
    </div>
    <div class="game-selector-footer">
      <el-button @click="close">稍后再说</el-button>
    </div>
  </div>
</template>

<script>
export default {
  name: 'GameSelector',
  emits: ['select-game', 'close'],
  methods: {
    selectGame(game) {
      this.$emit('select-game', game);
    },
    close() {
      this.$emit('close');
    }
  }
};
</script>

<style scoped>
.game-selector {
  padding: 30px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 12px;
  color: white;
  max-width: 600px;
  margin: 0 auto;
}

.game-selector-header {
  text-align: center;
  margin-bottom: 30px;
}

.game-selector-header h2 {
  margin: 0 0 10px 0;
  font-size: 36px;
  font-weight: bold;
}

.subtitle {
  margin: 0;
  font-size: 16px;
  opacity: 0.9;
}

.game-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
  margin-bottom: 30px;
}

.game-card {
  background: rgba(255, 255, 255, 0.2);
  backdrop-filter: blur(10px);
  border-radius: 12px;
  padding: 30px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s ease;
  border: 2px solid transparent;
}

.game-card:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: translateY(-5px);
  border-color: rgba(255, 255, 255, 0.5);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
}

.game-icon {
  font-size: 48px;
  margin-bottom: 15px;
  font-weight: bold;
}

.game-name {
  font-size: 24px;
  font-weight: bold;
  margin-bottom: 10px;
}

.game-desc {
  font-size: 14px;
  opacity: 0.9;
  line-height: 1.5;
}

.game-selector-footer {
  text-align: center;
}
</style>

