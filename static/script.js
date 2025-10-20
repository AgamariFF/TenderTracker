// Функции для выбора/очистки всех чекбоксов
function selectAll(className) {
    document.querySelectorAll('.' + className).forEach(checkbox => {
        checkbox.checked = true;
    });
}

function deselectAll(className) {
    document.querySelectorAll('.' + className).forEach(checkbox => {
        checkbox.checked = false;
    });
}

document.getElementById('searchForm').addEventListener('submit', function(e) {
    e.preventDefault();
    
    // Показываем индикатор загрузки
    document.getElementById('loadingSection').style.display = 'block';
    document.getElementById('resultSection').style.display = 'none';
    document.getElementById('errorSection').style.display = 'none';

    // Собираем данные формы
    const formData = new FormData(this);
    
    // Добавляем тип закупок (активные/завершенные)
    const procurementType = getProcurementType();
    formData.append('procurement_type', procurementType);
    
    // Добавляем булевы значения для переключателей
    formData.set('search_vent', document.getElementById('searchVent').checked);
    formData.set('search_doors', document.getElementById('searchDoors').checked);
    formData.set('search_build', document.getElementById('searchBuild').checked);
    formData.set('search_metal', document.getElementById('searchMetal').checked);

    // Добавляем минимальные суммы (только если переключатель активен и значение указано)
    if (document.getElementById('searchVent').checked) {
        const minPriceVent = document.getElementById('minPriceVent').value;
        if (minPriceVent && minPriceVent > 0) {
            formData.set('min_price_vent', minPriceVent);
        }
    }
    
    if (document.getElementById('searchDoors').checked) {
        const minPriceDoors = document.getElementById('minPriceDoors').value;
        if (minPriceDoors && minPriceDoors > 0) {
            formData.set('min_price_doors', minPriceDoors);
        }
    }
    
    if (document.getElementById('searchBuild').checked) {
        const minPriceBuild = document.getElementById('minPriceBuild').value;
        if (minPriceBuild && minPriceBuild > 0) {
            formData.set('min_price_build', minPriceBuild);
        }
    }
    
    if (document.getElementById('searchMetal').checked) {
        const minPriceMetal = document.getElementById('minPriceMetal').value;
        if (minPriceMetal && minPriceMetal > 0) {
            formData.set('min_price_metal', minPriceMetal);
        }
    }

    console.log('FormData содержимое:');
    for (let [key, value] of formData.entries()) {
        console.log(`${key}: ${value}`);
    }
    
    // Добавляем выбранные федеральные округа (customerPlace)
    const customerPlace = Array.from(document.querySelectorAll('.customer-place:checked'))
        .map(checkbox => checkbox.value);
    
    customerPlace.forEach(value => {
        formData.append('vent_customer_place', value);
    });

    // Добавляем выбранные коды регионов (delKladrIds)
    const delKladrIds = Array.from(document.querySelectorAll('.kladr-ids:checked'))
        .map(checkbox => checkbox.value);
    
    delKladrIds.forEach(value => {
        formData.append('vent_del_kladr_ids', value);
    });

    // Отправляем запрос
    fetch('/tender/searchTenders', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        document.getElementById('loadingSection').style.display = 'none';
        
        if (data.error) {
            showError(data.error);
        } else {
            showSuccess(data);
        }
    })
    .catch(error => {
        document.getElementById('loadingSection').style.display = 'none';
        showError('Ошибка сети: ' + error.message);
    });
});

function showSuccess(data) {
    document.getElementById('resultSection').style.display = 'block';

    const searchVent = document.getElementById('searchVent').checked
    const searchDoors = document.getElementById('searchDoors').checked
    const searchBuild = document.getElementById('searchBuild').checked
    const searchMetal = document.getElementById('searchMetal').checked
    
    const statsElement = document.getElementById('searchStats');
    
    if (data.stats !== undefined) {
        statsElement.style.display = 'block';
        
        let statsHTML = '<div class="row">';

        if (data.stats.totalFound !== undefined) {
            statsHTML += `
                <div class="col-md-4">
                    <div class="card bg-light">
                        <div class="card-body text-center">
                            <h4 class="text-success">${data.stats.totalFound}</h4>
                            <small class="text-muted">Всего найдено закупок</small>
                        </div>
                    </div>
                </div>
            `;
        }

        if (searchVent) {
            statsHTML += `
                <div class="col-md-4">
                    <div class="card bg-light">
                        <div class="card-body text-center">
                            <h4 class="text-primary">${data.stats.ventFound}</h4>
                            <small class="text-muted">Найдено закупок по вентиляции</small>
                        </div>
                    </div>
                </div>
            `;
        }

        if (searchDoors) {
            statsHTML += `
                <div class="col-md-4">
                    <div class="card bg-light">
                        <div class="card-body text-center">
                            <h4 class="text-primary">${data.stats.doorsFound}</h4>
                            <small class="text-muted">Найдено закупок по монтажу дверей</small>
                        </div>
                    </div>
                </div>
            `;
        }

        if (searchBuild) {
            statsHTML += `
                <div class="col-md-4">
                    <div class="card bg-light">
                        <div class="card-body text-center">
                            <h4 class="text-primary">${data.stats.buildFound}</h4>
                            <small class="text-muted">Найдено закупок по строительству/реконструкции</small>
                        </div>
                    </div>
                </div>
            `;
        }

        if (searchMetal) {
            statsHTML += `
                <div class="col-md-4">
                    <div class="card bg-light">
                        <div class="card-body text-center">
                            <h4 class="text-primary">${data.stats.metalFound}</h4>
                            <small class="text-muted">Найдено закупок по поставке металлоконструкций</small>
                        </div>
                    </div>
                </div>
            `;
        }

        statsHTML += '</div>';
        statsElement.innerHTML = statsHTML;
    } else {
        statsElement.style.display = 'none';
        statsElement.innerHTML = '';
    }
}

function showError(message) {
    document.getElementById('errorSection').style.display = 'block';
    document.getElementById('errorMessage').textContent = message;
}

function downloadFile() {
    // Создаем временную ссылку для скачивания
    const link = document.createElement('a');
    link.href = '/tender/download?filename=Закупки.xlsx&t=' + new Date().getTime();
    link.download = 'Закупки_' + new Date().toISOString().split('T')[0] + '.xlsx';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}

// Функция показа помощи
function showHelp() {
    const helpModal = new bootstrap.Modal(document.getElementById('helpModal'));
    helpModal.show();
}

function changePort(port, path = '') {
    const host = window.location.hostname;
    window.location.href = `//${host}:${port}/${path}`;
    if (path == '/') {
        window.location.href = `//${host}:${port}/`
    }
}

function setMinPrices(amount) {
    const priceInputs = [
        'minPriceVent', 'minPriceDoors', 'minPriceBuild', 'minPriceMetal'
    ];

    priceInputs.forEach(inputId => {
        const input = document.getElementById(inputId);
        if (input) {
            input.value = amount;
        }
    });
}

function clearMinPrices() {
    const priceInputs = [
        'minPriceVent', 'minPriceDoors', 'minPriceBuild', 'minPriceMetal'
    ];

    priceInputs.forEach(inputId => {
        const input = document.getElementById(inputId);
        if (input) {
            input.value = '';
        }
    });
}

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    // Управление состоянием полей ввода минимальных сумм
    const switches = ['searchVent', 'searchDoors', 'searchBuild', 'searchMetal'];

    switches.forEach(switchId => {
        const switchElement = document.getElementById(switchId);
        const priceInput = document.getElementById('minPrice' + switchId.charAt(6) + switchId.slice(7));
        
        if (switchElement && priceInput) {
            priceInput.disabled = !switchElement.checked;
            
            switchElement.addEventListener('change', function() {
                priceInput.disabled = !this.checked;
                if (!this.checked) {
                    priceInput.value = '';
                }
            });
        }
    });

    // Обработчики для переключателя типа закупок
    const activeRadio = document.getElementById('activeProcurements');
    const completedRadio = document.getElementById('completedProcurements');
    const activeHint = document.getElementById('activeProcurementsHint');
    const completedHint = document.getElementById('completedProcurementsHint');
    
    activeRadio.addEventListener('change', function() {
        if (this.checked) {
            activeHint.style.display = 'inline';
            completedHint.style.display = 'none';
            updateSearchButtonText('Найти активные закупки');
        }
    });
    
    completedRadio.addEventListener('change', function() {
        if (this.checked) {
            activeHint.style.display = 'none';
            completedHint.style.display = 'inline';
            updateSearchButtonText('Найти завершенные закупки');
        }
    });
    
    function updateSearchButtonText(text) {
        const searchButton = document.querySelector('#searchForm button[type="submit"]');
        const icon = '<i class="fas fa-search me-2"></i>';
        searchButton.innerHTML = icon + text;
    }
    
    // Инициализация текста кнопки при загрузке
    updateSearchButtonText('Найти активные закупки');
});

// Получение текущего выбранного типа закупок
function getProcurementType() {
    if (document.getElementById('completedProcurements').checked) {
        return 'completed';
    }
    return 'active';
}